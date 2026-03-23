var __defProp = Object.defineProperty;
var __defProps = Object.defineProperties;
var __getOwnPropDescs = Object.getOwnPropertyDescriptors;
var __getOwnPropSymbols = Object.getOwnPropertySymbols;
var __hasOwnProp = Object.prototype.hasOwnProperty;
var __propIsEnum = Object.prototype.propertyIsEnumerable;
var __defNormalProp = (obj, key, value) => key in obj ? __defProp(obj, key, { enumerable: true, configurable: true, writable: true, value }) : obj[key] = value;
var __spreadValues = (a, b) => {
  for (var prop in b || (b = {}))
    if (__hasOwnProp.call(b, prop))
      __defNormalProp(a, prop, b[prop]);
  if (__getOwnPropSymbols)
    for (var prop of __getOwnPropSymbols(b)) {
      if (__propIsEnum.call(b, prop))
        __defNormalProp(a, prop, b[prop]);
    }
  return a;
};
var __spreadProps = (a, b) => __defProps(a, __getOwnPropDescs(b));
var __objRest = (source, exclude) => {
  var target = {};
  for (var prop in source)
    if (__hasOwnProp.call(source, prop) && exclude.indexOf(prop) < 0)
      target[prop] = source[prop];
  if (source != null && __getOwnPropSymbols)
    for (var prop of __getOwnPropSymbols(source)) {
      if (exclude.indexOf(prop) < 0 && __propIsEnum.call(source, prop))
        target[prop] = source[prop];
    }
  return target;
};
var __async = (__this, __arguments, generator) => {
  return new Promise((resolve, reject) => {
    var fulfilled = (value) => {
      try {
        step(generator.next(value));
      } catch (e) {
        reject(e);
      }
    };
    var rejected = (value) => {
      try {
        step(generator.throw(value));
      } catch (e) {
        reject(e);
      }
    };
    var step = (x) => x.done ? resolve(x.value) : Promise.resolve(x.value).then(fulfilled, rejected);
    step((generator = generator.apply(__this, __arguments)).next());
  });
};

// src/index.ts
import React from "react";

// src/utils/loadImageURL.ts
var isDataURL = (str) => {
  const regex = /^\s*data:([a-z]+\/[a-z]+(;[a-z-]+=[a-z-]+)?)?(;base64)?,[a-z0-9!$&',()*+;=\-._~:@/?%\s]*\s*$/i;
  return !!str.match(regex);
};
var loadImageURL = (imageURL, crossOrigin) => new Promise((resolve, reject) => {
  const image = new Image();
  image.onload = () => resolve(image);
  image.onerror = reject;
  if (!isDataURL(imageURL) && crossOrigin) {
    image.crossOrigin = crossOrigin;
  }
  image.src = imageURL;
});

// src/utils/loadImageFile.ts
var loadImageFile = (file) => new Promise((resolve, reject) => {
  const reader = new FileReader();
  reader.onload = (e) => {
    var _a;
    try {
      if (!((_a = e == null ? void 0 : e.target) == null ? void 0 : _a.result)) {
        throw new Error("No image data");
      }
      const image = loadImageURL(e.target.result);
      resolve(image);
    } catch (e2) {
      reject(e2);
    }
  };
  reader.readAsDataURL(file);
});

// src/utils/isPassiveSupported.ts
var isPassiveSupported = () => {
  let passiveSupported = false;
  try {
    const options = Object.defineProperty({}, "passive", {
      get: function() {
        passiveSupported = true;
      }
    });
    const handler = () => {
    };
    window.addEventListener("test", handler, options);
    window.removeEventListener("test", handler, options);
  } catch (err) {
    passiveSupported = false;
  }
  return passiveSupported;
};

// src/utils/isTouchDevice.ts
var isTouchDevice = typeof window !== "undefined" && typeof navigator !== "undefined" && ("ontouchstart" in window || navigator.maxTouchPoints > 0);

// src/utils/isFileAPISupported.ts
var isFileAPISupported = typeof File !== "undefined";

// src/index.ts
var drawRoundedRect = (context, x, y, width, height, borderRadius) => {
  if (borderRadius === 0) {
    context.rect(x, y, width, height);
  } else {
    const widthMinusRad = width - borderRadius;
    const heightMinusRad = height - borderRadius;
    context.translate(x, y);
    context.arc(
      borderRadius,
      borderRadius,
      borderRadius,
      Math.PI,
      Math.PI * 1.5
    );
    context.lineTo(widthMinusRad, 0);
    context.arc(
      widthMinusRad,
      borderRadius,
      borderRadius,
      Math.PI * 1.5,
      Math.PI * 2
    );
    context.lineTo(width, heightMinusRad);
    context.arc(
      widthMinusRad,
      heightMinusRad,
      borderRadius,
      Math.PI * 2,
      Math.PI * 0.5
    );
    context.lineTo(borderRadius, height);
    context.arc(
      borderRadius,
      heightMinusRad,
      borderRadius,
      Math.PI * 0.5,
      Math.PI
    );
    context.closePath();
    context.translate(-x, -y);
  }
};
var drawGrid = (context, x, y, width, height, gridColor) => {
  context.fillStyle = gridColor;
  const thirdsX = width / 3;
  const thirdsY = height / 3;
  context.fillRect(x, y, 1, height);
  context.fillRect(thirdsX + x, y, 1, height);
  context.fillRect(thirdsX * 2 + x, y, 1, height);
  context.fillRect(thirdsX * 3 + x, y, 1, height);
  context.fillRect(thirdsX * 4 + x, y, 1, height);
  context.fillRect(x, y, width, 1);
  context.fillRect(x, thirdsY + y, width, 1);
  context.fillRect(x, thirdsY * 2 + y, width, 1);
  context.fillRect(x, thirdsY * 3 + y, width, 1);
  context.fillRect(x, thirdsY * 4 + y, width, 1);
};
var defaultEmptyImage = {
  x: 0.5,
  y: 0.5
};
var AvatarEditor = class extends React.Component {
  constructor() {
    super(...arguments);
    this.canvas = React.createRef();
    this.pixelRatio = typeof window !== "undefined" && window.devicePixelRatio ? window.devicePixelRatio : 1;
    this.state = {
      drag: false,
      my: void 0,
      mx: void 0,
      image: defaultEmptyImage
    };
    this.handleImageReady = (image) => {
      var _a, _b;
      const imageState = __spreadProps(__spreadValues({}, this.getInitialSize(image.width, image.height)), {
        resource: image,
        x: 0.5,
        y: 0.5
      });
      this.setState({ drag: false, image: imageState }, this.props.onImageReady);
      (_b = (_a = this.props).onLoadSuccess) == null ? void 0 : _b.call(_a, imageState);
    };
    this.clearImage = () => {
      const canvas = this.getCanvas();
      const context = this.getContext();
      context.clearRect(0, 0, canvas.width, canvas.height);
      this.setState({ image: defaultEmptyImage });
    };
    this.handleMouseDown = (e) => {
      e.preventDefault();
      this.setState({ drag: true, mx: void 0, my: void 0 });
    };
    this.handleTouchStart = (e) => {
      this.setState({ drag: true, mx: void 0, my: void 0 });
    };
    this.handleMouseUp = () => {
      var _a, _b;
      if (this.state.drag) {
        this.setState({ drag: false });
        (_b = (_a = this.props).onMouseUp) == null ? void 0 : _b.call(_a);
      }
    };
    this.handleMouseMove = (e) => {
      var _a, _b, _c, _d;
      if (!this.state.drag) {
        return;
      }
      e.preventDefault();
      const mousePositionX = "targetTouches" in e ? e.targetTouches[0].pageX : e.clientX;
      const mousePositionY = "targetTouches" in e ? e.targetTouches[0].pageY : e.clientY;
      this.setState({ mx: mousePositionX, my: mousePositionY });
      let rotate = this.props.rotate;
      rotate %= 360;
      rotate = rotate < 0 ? rotate + 360 : rotate;
      if (this.state.mx && this.state.my && this.state.image.width && this.state.image.height) {
        const mx = this.state.mx - mousePositionX;
        const my = this.state.my - mousePositionY;
        const width = this.state.image.width * this.props.scale;
        const height = this.state.image.height * this.props.scale;
        let { x: lastX, y: lastY } = this.getCroppingRect();
        lastX *= width;
        lastY *= height;
        const toRadians = (degree) => degree * (Math.PI / 180);
        const cos = Math.cos(toRadians(rotate));
        const sin = Math.sin(toRadians(rotate));
        const x = lastX + mx * cos + my * sin;
        const y = lastY + -mx * sin + my * cos;
        const relativeWidth = 1 / this.props.scale * this.getXScale();
        const relativeHeight = 1 / this.props.scale * this.getYScale();
        const position = {
          x: x / width + relativeWidth / 2,
          y: y / height + relativeHeight / 2
        };
        (_b = (_a = this.props).onPositionChange) == null ? void 0 : _b.call(_a, position);
        this.setState({ image: __spreadValues(__spreadValues({}, this.state.image), position) });
      }
      (_d = (_c = this.props).onMouseMove) == null ? void 0 : _d.call(_c, e);
    };
  }
  componentDidMount() {
    if (this.props.disableHiDPIScaling) {
      this.pixelRatio = 1;
    }
    const context = this.getContext();
    if (this.props.image) {
      this.loadImage(this.props.image);
    }
    this.paint(context);
    const options = isPassiveSupported() ? { passive: false } : false;
    document.addEventListener("mousemove", this.handleMouseMove, options);
    document.addEventListener("mouseup", this.handleMouseUp, options);
    if (isTouchDevice) {
      document.addEventListener("touchmove", this.handleMouseMove, options);
      document.addEventListener("touchend", this.handleMouseUp, options);
    }
  }
  componentDidUpdate(prevProps, prevState) {
    var _a, _b;
    if (this.props.image && (this.props.image !== prevProps.image || this.props.width !== prevProps.width || this.props.height !== prevProps.height || this.props.backgroundColor !== prevProps.backgroundColor)) {
      this.loadImage(this.props.image);
    } else if (!this.props.image && prevState.image !== defaultEmptyImage) {
      this.clearImage();
    }
    const context = this.getContext();
    context.clearRect(0, 0, this.getCanvas().width, this.getCanvas().height);
    this.paint(context);
    this.paintImage(context, this.state.image, this.props.border);
    if (prevProps.image !== this.props.image || prevProps.width !== this.props.width || prevProps.height !== this.props.height || prevProps.position !== this.props.position || prevProps.scale !== this.props.scale || prevProps.rotate !== this.props.rotate || prevState.my !== this.state.my || prevState.mx !== this.state.mx || prevState.image.x !== this.state.image.x || prevState.image.y !== this.state.image.y) {
      (_b = (_a = this.props).onImageChange) == null ? void 0 : _b.call(_a);
    }
  }
  getCanvas() {
    if (!this.canvas.current) {
      throw new Error(
        "No canvas found, please report this to: https://github.com/mosch/react-avatar-editor/issues"
      );
    }
    return this.canvas.current;
  }
  getContext() {
    const context = this.getCanvas().getContext("2d");
    if (!context) {
      throw new Error(
        "No context found, please report this to: https://github.com/mosch/react-avatar-editor/issues"
      );
    }
    return context;
  }
  componentWillUnmount() {
    document.removeEventListener("mousemove", this.handleMouseMove, false);
    document.removeEventListener("mouseup", this.handleMouseUp, false);
    if (isTouchDevice) {
      document.removeEventListener("touchmove", this.handleMouseMove, false);
      document.removeEventListener("touchend", this.handleMouseUp, false);
    }
  }
  isVertical() {
    return !this.props.disableCanvasRotation && this.props.rotate % 180 !== 0;
  }
  getBorders(border = this.props.border) {
    return Array.isArray(border) ? border : [border, border];
  }
  getDimensions() {
    const { width, height, rotate, border } = this.props;
    const canvas = { width: 0, height: 0 };
    const [borderX, borderY] = this.getBorders(border);
    if (this.isVertical()) {
      canvas.width = height;
      canvas.height = width;
    } else {
      canvas.width = width;
      canvas.height = height;
    }
    canvas.width += borderX * 2;
    canvas.height += borderY * 2;
    return {
      canvas,
      rotate,
      width,
      height,
      border
    };
  }
  getImage() {
    const cropRect = this.getCroppingRect();
    const image = this.state.image;
    if (!image.resource) {
      throw new Error(
        "No image resource available, please report this to: https://github.com/mosch/react-avatar-editor/issues"
      );
    }
    cropRect.x *= image.resource.width;
    cropRect.y *= image.resource.height;
    cropRect.width *= image.resource.width;
    cropRect.height *= image.resource.height;
    const canvas = document.createElement("canvas");
    if (this.isVertical()) {
      canvas.width = cropRect.height;
      canvas.height = cropRect.width;
    } else {
      canvas.width = cropRect.width;
      canvas.height = cropRect.height;
    }
    const context = canvas.getContext("2d");
    if (!context) {
      throw new Error(
        "No context found, please report this to: https://github.com/mosch/react-avatar-editor/issues"
      );
    }
    context.translate(canvas.width / 2, canvas.height / 2);
    context.rotate(this.props.rotate * Math.PI / 180);
    context.translate(-(canvas.width / 2), -(canvas.height / 2));
    if (this.isVertical()) {
      context.translate(
        (canvas.width - canvas.height) / 2,
        (canvas.height - canvas.width) / 2
      );
    }
    if (this.props.backgroundColor) {
      context.fillStyle = this.props.backgroundColor;
      context.fillRect(0, 0, canvas.width, canvas.height);
    }
    context.drawImage(image.resource, -cropRect.x, -cropRect.y);
    return canvas;
  }
  /**
   * Get the image scaled to original canvas size.
   * This was default in 4.x and is now kept as a legacy method.
   */
  getImageScaledToCanvas() {
    const { width, height } = this.getDimensions();
    const canvas = document.createElement("canvas");
    if (this.isVertical()) {
      canvas.width = height;
      canvas.height = width;
    } else {
      canvas.width = width;
      canvas.height = height;
    }
    this.paintImage(canvas.getContext("2d"), this.state.image, 0, 1);
    return canvas;
  }
  getXScale() {
    if (!this.state.image.width || !this.state.image.height)
      throw new Error("Image dimension is unknown.");
    const canvasAspect = this.props.width / this.props.height;
    const imageAspect = this.state.image.width / this.state.image.height;
    return Math.min(1, canvasAspect / imageAspect);
  }
  getYScale() {
    if (!this.state.image.width || !this.state.image.height)
      throw new Error("Image dimension is unknown.");
    const canvasAspect = this.props.height / this.props.width;
    const imageAspect = this.state.image.height / this.state.image.width;
    return Math.min(1, canvasAspect / imageAspect);
  }
  getCroppingRect() {
    const position = this.props.position || {
      x: this.state.image.x,
      y: this.state.image.y
    };
    const width = 1 / this.props.scale * this.getXScale();
    const height = 1 / this.props.scale * this.getYScale();
    const croppingRect = {
      x: position.x - width / 2,
      y: position.y - height / 2,
      width,
      height
    };
    let xMin = 0;
    let xMax = 1 - croppingRect.width;
    let yMin = 0;
    let yMax = 1 - croppingRect.height;
    const isLargerThanImage = this.props.disableBoundaryChecks || width > 1 || height > 1;
    if (isLargerThanImage) {
      xMin = -croppingRect.width;
      xMax = 1;
      yMin = -croppingRect.height;
      yMax = 1;
    }
    return __spreadProps(__spreadValues({}, croppingRect), {
      x: Math.max(xMin, Math.min(croppingRect.x, xMax)),
      y: Math.max(yMin, Math.min(croppingRect.y, yMax))
    });
  }
  loadImage(file) {
    return __async(this, null, function* () {
      var _a, _b, _c, _d;
      if (isFileAPISupported && file instanceof File) {
        try {
          const image = yield loadImageFile(file);
          this.handleImageReady(image);
        } catch (error) {
          (_b = (_a = this.props).onLoadFailure) == null ? void 0 : _b.call(_a);
        }
      } else if (typeof file === "string") {
        try {
          const image = yield loadImageURL(file, this.props.crossOrigin);
          this.handleImageReady(image);
        } catch (e) {
          (_d = (_c = this.props).onLoadFailure) == null ? void 0 : _d.call(_c);
        }
      }
    });
  }
  getInitialSize(width, height) {
    let newHeight;
    let newWidth;
    const dimensions = this.getDimensions();
    const canvasRatio = dimensions.height / dimensions.width;
    const imageRatio = height / width;
    if (canvasRatio > imageRatio) {
      newHeight = dimensions.height;
      newWidth = Math.round(width * (newHeight / height));
    } else {
      newWidth = dimensions.width;
      newHeight = Math.round(height * (newWidth / width));
    }
    return {
      height: newHeight,
      width: newWidth
    };
  }
  paintImage(context, image, border, scaleFactor = this.pixelRatio) {
    if (!image.resource)
      return;
    const position = this.calculatePosition(image, border);
    context.save();
    context.translate(context.canvas.width / 2, context.canvas.height / 2);
    context.rotate(this.props.rotate * Math.PI / 180);
    context.translate(-(context.canvas.width / 2), -(context.canvas.height / 2));
    if (this.isVertical()) {
      context.translate(
        (context.canvas.width - context.canvas.height) / 2,
        (context.canvas.height - context.canvas.width) / 2
      );
    }
    context.scale(scaleFactor, scaleFactor);
    context.globalCompositeOperation = "destination-over";
    context.drawImage(
      image.resource,
      position.x,
      position.y,
      position.width,
      position.height
    );
    if (this.props.backgroundColor) {
      context.fillStyle = this.props.backgroundColor;
      context.fillRect(0, 0, context.canvas.width, context.canvas.height);
    }
    context.restore();
  }
  calculatePosition(image = this.state.image, border) {
    const [borderX, borderY] = this.getBorders(border);
    if (!image.width || !image.height) {
      throw new Error("Image dimension is unknown.");
    }
    const croppingRect = this.getCroppingRect();
    const width = image.width * this.props.scale;
    const height = image.height * this.props.scale;
    let x = -croppingRect.x * width;
    let y = -croppingRect.y * height;
    if (this.isVertical()) {
      x += borderY;
      y += borderX;
    } else {
      x += borderX;
      y += borderY;
    }
    return { x, y, height, width };
  }
  paint(context) {
    context.save();
    context.scale(this.pixelRatio, this.pixelRatio);
    context.translate(0, 0);
    context.fillStyle = "rgba(" + this.props.color.slice(0, 4).join(",") + ")";
    let borderRadius = this.props.borderRadius;
    const dimensions = this.getDimensions();
    const [borderSizeX, borderSizeY] = this.getBorders(dimensions.border);
    const height = dimensions.canvas.height;
    const width = dimensions.canvas.width;
    borderRadius = Math.max(borderRadius, 0);
    borderRadius = Math.min(
      borderRadius,
      width / 2 - borderSizeX,
      height / 2 - borderSizeY
    );
    context.beginPath();
    drawRoundedRect(
      context,
      borderSizeX,
      borderSizeY,
      width - borderSizeX * 2,
      height - borderSizeY * 2,
      borderRadius
    );
    context.rect(width, 0, -width, height);
    context.fill("evenodd");
    if (this.props.borderColor) {
      context.strokeStyle = "rgba(" + this.props.borderColor.slice(0, 4).join(",") + ")";
      context.lineWidth = 1;
      context.beginPath();
      drawRoundedRect(
        context,
        borderSizeX + 0.5,
        borderSizeY + 0.5,
        width - borderSizeX * 2 - 1,
        height - borderSizeY * 2 - 1,
        borderRadius
      );
      context.stroke();
    }
    if (this.props.showGrid) {
      drawGrid(
        context,
        borderSizeX,
        borderSizeY,
        width - borderSizeX * 2,
        height - borderSizeY * 2,
        this.props.gridColor
      );
    }
    context.restore();
  }
  render() {
    const _a = this.props, {
      scale,
      rotate,
      image,
      border,
      borderRadius,
      width,
      height,
      position,
      color,
      backgroundColor,
      style,
      crossOrigin,
      onLoadFailure,
      onLoadSuccess,
      onImageReady,
      onImageChange,
      onMouseUp,
      onMouseMove,
      onPositionChange,
      disableBoundaryChecks,
      disableHiDPIScaling,
      disableCanvasRotation,
      showGrid,
      gridColor,
      borderColor
    } = _a, rest = __objRest(_a, [
      "scale",
      "rotate",
      "image",
      "border",
      "borderRadius",
      "width",
      "height",
      "position",
      "color",
      "backgroundColor",
      "style",
      "crossOrigin",
      "onLoadFailure",
      "onLoadSuccess",
      "onImageReady",
      "onImageChange",
      "onMouseUp",
      "onMouseMove",
      "onPositionChange",
      "disableBoundaryChecks",
      "disableHiDPIScaling",
      "disableCanvasRotation",
      "showGrid",
      "gridColor",
      "borderColor"
    ]);
    const dimensions = this.getDimensions();
    const defaultStyle = {
      width: dimensions.canvas.width,
      height: dimensions.canvas.height,
      cursor: this.state.drag ? "grabbing" : "grab",
      touchAction: "none"
    };
    const attributes = {
      width: dimensions.canvas.width * this.pixelRatio,
      height: dimensions.canvas.height * this.pixelRatio,
      onMouseDown: this.handleMouseDown,
      onTouchStart: this.handleTouchStart,
      style: __spreadValues(__spreadValues({}, defaultStyle), style)
    };
    return React.createElement("canvas", __spreadProps(__spreadValues(__spreadValues({}, attributes), rest), {
      ref: this.canvas
    }));
  }
};
AvatarEditor.defaultProps = {
  scale: 1,
  rotate: 0,
  border: 25,
  borderRadius: 0,
  width: 200,
  height: 200,
  color: [0, 0, 0, 0.5],
  showGrid: false,
  gridColor: "#666",
  disableBoundaryChecks: false,
  disableHiDPIScaling: false,
  disableCanvasRotation: true
};
var src_default = AvatarEditor;
export {
  src_default as default
};
//# sourceMappingURL=index.mjs.map